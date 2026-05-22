counter: int = 0
labels: list[str] = []


def add(name: str) -> None:
    global counter
    counter += 1
    labels.append(name)


def main() -> None:
    add("a")
    add("b")
    add("c")
    print(counter)
    print(labels)


if __name__ == "__main__":
    main()
