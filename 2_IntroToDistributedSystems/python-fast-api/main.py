from fastapi import FastAPI, Depends, HTTPException, status
from fastapi.security import OAuth2PasswordBearer
from pydantic import BaseModel
from datetime import datetime, timedelta
from jose import JWTError, jwt
from typing import Optional, List
import bcrypt
import asyncpg
from asyncpg import Pool
from typing import Any, List, Optional

# PostgreSQL settings
DATABASE_URL = "postgres://fastapi_user:fastapi_pass@pgbouncer:5432/fastapi_db"
SECRET_KEY = "09d25e094faa6ca2556c818166b7a9563b93f7099f6f0f4caa6cf63b88e8d3e7"
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 120  # 2 hours

app = FastAPI()

# Database connection pool
class UninitializedDatabasePoolError(Exception):
    def __init__(
        self,
        message="The database connection pool has not been properly initialized. Please ensure setup is called.",
    ):
        self.message = message
        super().__init__(self.message)


class DataBasePool:
    _db_pool: Optional[Pool] = None

    @classmethod
    async def setup(cls, timeout: Optional[float] = None):
        cls._db_pool = await asyncpg.create_pool(
            dsn=DATABASE_URL,  # Use the DATABASE_URL from settings
            min_size=10,
            max_size=30,
            timeout=60,
        )
        cls._timeout = timeout

    @classmethod
    async def get_pool(cls):
        if not cls._db_pool:
            raise UninitializedDatabasePoolError()
        return cls._db_pool

    @classmethod
    async def teardown(cls):
        if not cls._db_pool:
            raise UninitializedDatabasePoolError()
        await cls._db_pool.close()


async def execute_query_with_pool(
    query: str, *args: Any, fetch: bool = False, fetch_one: bool = False
) -> Optional[List[Any]]:
    db_pool: Pool = await DataBasePool.get_pool()
    async with db_pool.acquire() as connection:
        async with connection.transaction():
            if fetch:
                result = await connection.fetch(query, *args)
            elif fetch_one:
                result = await connection.fetchrow(query, *args)
            else:
                result = await connection.execute(query, *args)
            return result


# Security
oauth2_scheme = OAuth2PasswordBearer(tokenUrl="token")


# Models
class UserBase(BaseModel):
    username: str
    full_name: Optional[str] = None
    email: Optional[str] = None
    disabled: Optional[bool] = None
    scopes: Optional[List[str]] = []


class UserCreate(UserBase):
    password: str


class UserResponse(UserBase):
    id: int

    class Config:
        from_attributes = True  # Updated from orm_mode


class Token(BaseModel):
    access_token: str
    token_type: str


class TokenData(BaseModel):
    username: Optional[str] = None
    scopes: List[str] = []


# Helper Functions
async def get_user(username: str):
    query = "SELECT * FROM users WHERE username = $1"
    return await execute_query_with_pool(query, username, fetch_one=True)


async def verify_password(plain_password: str, hashed_password: str) -> bool:
    return bcrypt.checkpw(plain_password.encode("utf-8"), hashed_password.encode("utf-8"))


async def authenticate_user(username: str, password: str):
    user = await get_user(username)
    if user and await verify_password(password, user["hashed_password"]):
        return user
    return None


def create_access_token(data: dict, expires_delta: Optional[timedelta] = None):
    to_encode = data.copy()
    expire = datetime.utcnow() + (expires_delta or timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES))
    to_encode.update({"exp": expire})
    return jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)


async def get_current_user(token: str = Depends(oauth2_scheme)):
    credentials_exception = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Could not validate credentials",
        headers={"WWW-Authenticate": "Bearer"},
    )
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
        username: str = payload.get("sub")
        if username is None:
            raise credentials_exception
        token_data = TokenData(username=username, scopes=payload.get("scopes", []))
    except JWTError:
        raise credentials_exception

    user = await get_user(token_data.username)
    if not user:
        raise credentials_exception
    return user


async def get_current_active_user(current_user: dict = Depends(get_current_user)):
    if current_user["disabled"]:
        raise HTTPException(status_code=400, detail="Inactive user")
    return current_user


# Routes
@app.post("/token", response_model=Token)
async def login_for_access_token(username: str, password: str, scopes: Optional[List[str]] = None):
    user = await authenticate_user(username, password)
    if not user:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Incorrect username or password",
            headers={"WWW-Authenticate": "Bearer"},
        )
    access_token_expires = timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES)
    access_token = create_access_token(
        data={"sub": user["username"], "scopes": scopes or user["scopes"]}, expires_delta=access_token_expires
    )
    return {"access_token": access_token, "token_type": "bearer"}

@app.get("/check")
async def check_token(token: str = Depends(oauth2_scheme)):
    credentials_exception = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Could not validate credentials",
        headers={"WWW-Authenticate": "Bearer"},
    )
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
        username: str = payload.get("sub")
        if username is None:
            raise credentials_exception
        token_data = TokenData(username=username, scopes=payload.get("scopes", []))
    except JWTError:
        raise credentials_exception

    return {"scopes": token_data.scopes}

@app.post("/users/", response_model=UserResponse)
async def create_user(user: UserCreate):
    existing_user = await get_user(user.username)
    if existing_user:
        raise HTTPException(status_code=400, detail="Username already registered")
    hashed_password = bcrypt.hashpw(user.password.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")
    query = """
    INSERT INTO users (username, full_name, email, hashed_password, disabled, scopes)
    VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id
    """
    values = [
        user.username,
        user.full_name,
        user.email,
        hashed_password,
        user.disabled or False,
        user.scopes or [],
    ]
    user_id = await execute_query_with_pool(query, *values, fetch_one=True)
    return {"id": user_id["id"], **user.dict(exclude={"password"})}


# Startup and Shutdown Events
@app.on_event("startup")
async def startup():
    await DataBasePool.setup()
    # Ensure the users table exists
    query = """
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        username VARCHAR(255) UNIQUE NOT NULL,
        full_name VARCHAR(255),
        email VARCHAR(255) UNIQUE,
        hashed_password VARCHAR(255) NOT NULL,
        disabled BOOLEAN DEFAULT FALSE,
        scopes TEXT[]
    );
    """
    await execute_query_with_pool(query)


@app.on_event("shutdown")
async def shutdown():
    await DataBasePool.teardown()


# Run the application
if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)