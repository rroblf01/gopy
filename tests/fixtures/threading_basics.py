import threading


def main() -> None:
    print(threading.active_count() > 0)
    print(threading.get_ident() > 0)


if __name__ == "__main__":
    main()
