import socket


def main() -> None:
    p = socket.inet_pton(socket.AF_INET, "127.0.0.1")
    print(len(p))
    s = socket.inet_ntop(socket.AF_INET, p)
    print(s)


if __name__ == "__main__":
    main()
