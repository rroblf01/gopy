import threading


def main() -> None:
    lock = threading.Lock()
    print(lock.locked())
    lock.acquire()
    print(lock.locked())
    lock.release()
    print(lock.locked())
    r = threading.RLock()
    r.acquire()
    print(r.locked())
    r.release()
    print(r.locked())


if __name__ == "__main__":
    main()
