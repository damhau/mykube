from datetime import datetime, timedelta, timezone

import pytest

from mykube_relay.errors import (
    InvalidCode,
    MaxPairAttemptsExceeded,
    SessionExpired,
    SessionNotFound,
)
from mykube_relay.models import SessionStatus
from mykube_relay.session_store import SessionStore


async def test_create_session():
    store = SessionStore()
    session = await store.create_session()
    assert len(session.code) == 6
    assert session.code.isdigit()
    assert session.status == SessionStatus.WAITING
    assert session.session_id in store._sessions


async def test_pair_session():
    store = SessionStore()
    session = await store.create_session()
    code = session.code

    paired = await store.pair_session(code)
    assert paired.status == SessionStatus.PAIRED
    assert paired.paired_at is not None
    assert code not in store._code_index


async def test_pair_invalid_code():
    store = SessionStore()
    with pytest.raises(InvalidCode):
        await store.pair_session("000000")


async def test_pair_already_paired():
    store = SessionStore()
    session = await store.create_session()
    await store.pair_session(session.code)

    # code is removed after pairing, so this raises InvalidCode
    with pytest.raises(InvalidCode):
        await store.pair_session(session.code)


async def test_pair_expired_session():
    store = SessionStore()
    session = await store.create_session()
    # Force expiration
    session.expires_at = datetime.now(timezone.utc) - timedelta(seconds=1)

    with pytest.raises(SessionExpired):
        await store.pair_session(session.code)


async def test_max_pair_attempts():
    store = SessionStore()
    session = await store.create_session()
    code = session.code

    # Manually set attempts near limit
    from mykube_relay.config import settings
    session.pair_attempts = settings.MAX_PAIR_ATTEMPTS

    with pytest.raises(MaxPairAttemptsExceeded):
        await store.pair_session(code)


async def test_get_session():
    store = SessionStore()
    session = await store.create_session()
    fetched = await store.get_session(session.session_id)
    assert fetched.session_id == session.session_id


async def test_get_session_not_found():
    store = SessionStore()
    with pytest.raises(SessionNotFound):
        await store.get_session("nonexistent")


async def test_get_session_expired():
    store = SessionStore()
    session = await store.create_session()
    session.expires_at = datetime.now(timezone.utc) - timedelta(seconds=1)

    with pytest.raises(SessionExpired):
        await store.get_session(session.session_id)


async def test_remove_session():
    store = SessionStore()
    session = await store.create_session()
    await store.remove_session(session.session_id)
    assert session.session_id not in store._sessions
    assert session.code not in store._code_index


async def test_cleanup_expired():
    store = SessionStore()
    s1 = await store.create_session()
    s2 = await store.create_session()

    # Expire s1
    s1.expires_at = datetime.now(timezone.utc) - timedelta(seconds=1)

    await store.cleanup_expired()
    assert s1.session_id not in store._sessions
    assert s2.session_id in store._sessions
