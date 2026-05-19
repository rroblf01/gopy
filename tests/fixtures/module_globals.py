counter: int = 0
name: str = "init"


def inc() -> None:
    global counter
    counter = counter + 1


def rename(s: str) -> None:
    global name
    name = s


def read_counter() -> int:
    return counter


def main() -> None:
    inc()
    inc()
    inc()
    rename("after")
    print(counter)
    print(read_counter())
    print(name)


if __name__ == "__main__":
    main()
