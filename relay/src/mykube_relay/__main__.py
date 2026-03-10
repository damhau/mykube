import uvicorn

from .config import settings

uvicorn.run(
    "mykube_relay.app:app",
    host=settings.HOST,
    port=settings.PORT,
    ws_ping_interval=60,
    ws_ping_timeout=60,
)
