from collections import deque


def main() -> None:
    d = deque([1, 2, 3])
    d.append(4)
    d.appendleft(0)
    # Deque iteration / len match Python order: leftmost first.
    print(d.popleft())
    print(d.pop())
    # Remaining: [1, 2, 3]
    print(d.popleft())
    print(d.popleft())
    print(d.popleft())


if __name__ == "__main__":
    main()
