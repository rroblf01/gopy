import socket


def main() -> None:
    ip: str = socket.gethostbyname("localhost")
    print(ip in ["127.0.0.1", "::1"])


if __name__ == "__main__":
    main()
