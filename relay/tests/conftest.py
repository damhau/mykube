import pytest
from httpx import ASGITransport, AsyncClient

from mykube_relay.app import app, store


@pytest.fixture(autouse=True)
async def _clear_store():
    """Clear the session store before each test."""
    store._sessions.clear()
    store._code_index.clear()
    yield
    store._sessions.clear()
    store._code_index.clear()


@pytest.fixture
def session_store():
    return store


@pytest.fixture
async def client():
    transport = ASGITransport(app=app)
    async with AsyncClient(transport=transport, base_url="http://test") as ac:
        yield ac
