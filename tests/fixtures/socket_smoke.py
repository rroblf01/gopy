import socket


def main() -> None:
    print(socket.AF_INET == 2)
    print(socket.SOCK_STREAM == 1)
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.close()
    print("ok")
    try:
        bad = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        bad.connect(("127.0.0.1", 1))
        bad.close()
    except Exception:
        print("refused")


if __name__ == "__main__":
    main()
