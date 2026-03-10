from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    SESSION_TTL_WAITING: int = 120
    SESSION_TTL_PAIRED: int = 3600
    MAX_PAIR_ATTEMPTS: int = 5
    CLEANUP_INTERVAL: int = 10
    HOST: str = "0.0.0.0"
    PORT: int = 8000


settings = Settings()
