class User:
    def __init__(self, name: str, age: int) -> None:
        self.name = name
        self.age = age


def main() -> None:
    u: User = User("ada", 36)
    # getattr / setattr / hasattr dispatch through generated per-class
    # accessor helpers. Field types come from the declared annotations.
    print(getattr(u, "name"))
    print(getattr(u, "age"))
    print(getattr(u, "missing", "fallback"))
    print(hasattr(u, "name"))
    print(hasattr(u, "missing"))
    setattr(u, "age", 37)
    print(u.age)


if __name__ == "__main__":
    main()
