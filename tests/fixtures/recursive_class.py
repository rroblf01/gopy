from typing import Optional


class Tree:
    value: int
    left: Optional["Tree"]
    right: Optional["Tree"]

    def __init__(self, value: int) -> None:
        self.value = value
        self.left = None
        self.right = None

    def insert(self, v: int) -> None:
        if v < self.value:
            if self.left is None:
                self.left = Tree(v)
            else:
                self.left.insert(v)
        else:
            if self.right is None:
                self.right = Tree(v)
            else:
                self.right.insert(v)

    def inorder(self) -> list[int]:
        out: list[int] = []
        if self.left is not None:
            out.extend(self.left.inorder())
        out.append(self.value)
        if self.right is not None:
            out.extend(self.right.inorder())
        return out


def main() -> None:
    t = Tree(5)
    for v in [3, 7, 1, 4, 6, 8, 2]:
        t.insert(v)
    print(t.inorder())


if __name__ == "__main__":
    main()
