class Stack:
    def __init__(self) -> None:
        self.items: list[int] = []

    def push(self, v: int) -> None:
        self.items.append(v)

    def pop(self) -> int:
        v = self.items[-1]
        self.items = self.items[:-1]
        return v

    def size(self) -> int:
        return len(self.items)


def make_stack(*vals: int) -> Stack:
    s = Stack()
    for v in vals:
        s.push(v)
    return s


def main() -> None:
    s = make_stack(1, 2, 3, 4)
    print(s.size())
    print(s.pop())
    print(s.pop())
    print(s.size())


if __name__ == "__main__":
    main()
