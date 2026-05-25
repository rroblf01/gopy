import queue


def main() -> None:
    q = queue.Queue()
    q.put("a")
    q.put("b")
    print(q.get())
    print(q.get())


if __name__ == "__main__":
    main()
