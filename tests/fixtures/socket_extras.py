import socket


def main() -> None:
    print(socket.IPPROTO_TCP == 6)
    print(socket.IPPROTO_UDP == 17)
    print(socket.TCP_NODELAY == 1)
    print(socket.SOCK_RAW == 3)
    print(socket.SOMAXCONN >= 1)


if __name__ == "__main__":
    main()
