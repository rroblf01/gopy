class Stack:
    items: list[int]

    def __init__(self) -> None:
        self.items = []

    def push(self, v: int) -> None:
        self.items.append(v)

    def pop(self) -> int:
        v = self.items[-1]
        self.items = self.items[:-1]
        return v


def main() -> None:
    s = Stack()
    s.push(1)
    s.push(2)
    print(s.pop())


if __name__ == "__main__":
    main()
