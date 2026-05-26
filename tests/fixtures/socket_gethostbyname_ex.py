import socket


def main() -> None:
    res = socket.gethostbyname_ex("localhost")
    print(len(res))


if __name__ == "__main__":
    main()
