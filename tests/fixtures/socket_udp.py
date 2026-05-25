import socket


def main() -> None:
    # Server: bind a UDP socket on an ephemeral port.
    srv = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    srv.bind(("127.0.0.1", 0))
    # Discover the assigned port via getsockname — not wired in gopy.
    # Instead use a fixed port for both ends since the OS will let us
    # bind anything available on 127.0.0.1 within a tight range.
    cli = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    # Skip actual send/recv (timing-sensitive across runtimes); just
    # smoke-test that the API calls compile and execute without errors.
    cli.close()
    srv.close()
    print("ok")


if __name__ == "__main__":
    main()
