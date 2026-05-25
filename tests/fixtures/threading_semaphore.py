import threading


def main() -> None:
    s = threading.Semaphore(2)
    s.acquire()
    s.acquire()
    s.release()
    s.release()
    print("ok")


if __name__ == "__main__":
    main()
