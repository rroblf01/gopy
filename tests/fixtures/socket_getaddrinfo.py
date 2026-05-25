import socket


def main() -> None:
    infos = socket.getaddrinfo("127.0.0.1", 80)
    # Smoke: at least one entry expected for loopback.
    print(len(infos) >= 1)


if __name__ == "__main__":
    main()
