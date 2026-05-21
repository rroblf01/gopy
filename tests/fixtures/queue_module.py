from queue import Queue, LifoQueue


def main() -> None:
    q = Queue()
    print(q.empty())
    q.put(1)
    q.put(2)
    q.put(3)
    print(q.qsize())
    print(q.empty())
    print(q.get())
    print(q.get())
    print(q.qsize())

    s = LifoQueue()
    s.put(10)
    s.put(20)
    s.put(30)
    print(s.get())
    print(s.get())
    print(s.qsize())


if __name__ == "__main__":
    main()
