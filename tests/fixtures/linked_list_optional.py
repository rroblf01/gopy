from typing import Optional


class Node:
    def __init__(self, val: int) -> None:
        self.val = val
        self.next: Optional["Node"] = None


class LinkedList:
    def __init__(self) -> None:
        self.head: Optional[Node] = None

    def push(self, v: int) -> None:
        node = Node(v)
        node.next = self.head
        self.head = node

    def to_list(self) -> list[int]:
        out: list[int] = []
        cur = self.head
        while cur is not None:
            out.append(cur.val)
            cur = cur.next
        return out


def main() -> None:
    ll = LinkedList()
    ll.push(1)
    ll.push(2)
    ll.push(3)
    print(ll.to_list())


if __name__ == "__main__":
    main()
