import pytest
from starlette.testclient import TestClient

from mykube_relay.app import app, store


def test_agent_ws_connect():
    """Test that an agent can connect via WebSocket."""
    with TestClient(app) as tc:
        # Create a session via API
        resp = tc.post("/api/sessions")
        assert resp.status_code == 200
        session_id = resp.json()["session_id"]

        with tc.websocket_connect(f"/ws/agent/{session_id}") as ws:
            # Agent is connected, just verify the connection works
            # We can't easily test the full bridge without async
            pass


def test_client_ws_requires_pairing():
    """Test that a client cannot connect to an unpaired session."""
    with TestClient(app) as tc:
        resp = tc.post("/api/sessions")
        session_id = resp.json()["session_id"]

        # Client tries to connect before pairing
        with pytest.raises(Exception):
            with tc.websocket_connect(f"/ws/client/{session_id}") as ws:
                pass


def test_ws_bridge():
    """Test full WebSocket bridging between agent and client."""
    with TestClient(app) as tc:
        # Create session
        resp = tc.post("/api/sessions")
        data = resp.json()
        session_id = data["session_id"]
        code = data["code"]

        # Agent connects
        with tc.websocket_connect(f"/ws/agent/{session_id}") as agent_ws:
            # Pair the session
            pair_resp = tc.post("/api/pair", json={"code": code})
            assert pair_resp.status_code == 200

            # Client connects
            with tc.websocket_connect(f"/ws/client/{session_id}") as client_ws:
                # Agent receives "paired" signal first
                signal = agent_ws.receive_text()
                assert signal == "paired"

                # Client sends a "new:1" control signal to open a connection
                client_ws.send_text("new:1")
                msg = agent_ws.receive_text()
                assert msg == "new:1"

                # Client sends binary-framed data: [4-byte connID=1][payload]
                conn_id = (1).to_bytes(4, "big")
                client_ws.send_bytes(conn_id + b"hello from client")
                raw = agent_ws.receive_bytes()
                assert raw == conn_id + b"hello from client"

                # Agent sends binary-framed data back
                agent_ws.send_bytes(conn_id + b"hello from agent")
                raw = client_ws.receive_bytes()
                assert raw == conn_id + b"hello from agent"

                # Agent signals done for connection 1
                agent_ws.send_text("done:1")
                msg = client_ws.receive_text()
                assert msg == "done:1"


def test_ws_session_not_found():
    """Test WebSocket connection to nonexistent session."""
    with TestClient(app) as tc:
        with pytest.raises(Exception):
            with tc.websocket_connect("/ws/agent/nonexistent-id") as ws:
                pass
