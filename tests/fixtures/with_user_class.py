class Resource:
    def __init__(self, name: str) -> None:
        self.name = name

    def __enter__(self) -> "Resource":
        print(f"open {self.name}")
        return self

    def __exit__(self, exc_type, exc, tb) -> bool:
        print(f"close {self.name}")
        return False

    def use(self, x: int) -> None:
        print(f"{self.name}: {x}")


def main() -> None:
    with Resource("R1") as r:
        r.use(1)
        r.use(2)
    print("done")


if __name__ == "__main__":
    main()
