import functools


@functools.total_ordering
class Box:
    def __init__(self, n: int) -> None:
        self.n = n

    def __lt__(self, other: "Box") -> bool:
        return self.n < other.n

    def __eq__(self, other: "Box") -> bool:
        return self.n == other.n


def main() -> None:
    b = Box(5)
    print(b.n)
    print(len(functools.WRAPPER_ASSIGNMENTS) > 0)


if __name__ == "__main__":
    main()
