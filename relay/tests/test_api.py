import pytest


async def test_health(client):
    resp = await client.get("/api/health")
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}


async def test_create_session(client):
    resp = await client.post("/api/sessions")
    assert resp.status_code == 200
    data = resp.json()
    assert "session_id" in data
    assert "code" in data
    assert len(data["code"]) == 6


async def test_pair_session(client):
    # Create a session first
    create_resp = await client.post("/api/sessions")
    code = create_resp.json()["code"]

    # Pair with the code
    pair_resp = await client.post("/api/pair", json={"code": code})
    assert pair_resp.status_code == 200
    data = pair_resp.json()
    assert "session_id" in data


async def test_pair_invalid_code(client):
    resp = await client.post("/api/pair", json={"code": "999999"})
    assert resp.status_code == 400
    assert resp.json()["error"] == "invalid code"


async def test_pair_twice(client):
    create_resp = await client.post("/api/sessions")
    code = create_resp.json()["code"]

    # First pair succeeds
    pair_resp = await client.post("/api/pair", json={"code": code})
    assert pair_resp.status_code == 200

    # Second pair fails (code removed after pairing)
    pair_resp2 = await client.post("/api/pair", json={"code": code})
    assert pair_resp2.status_code == 400
