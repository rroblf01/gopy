import threading


def main() -> None:
    e = threading.Event()
    print(e.is_set())
    e.set()
    print(e.is_set())
    print(e.wait(0.1))
    e.clear()
    print(e.is_set())


if __name__ == "__main__":
    main()
