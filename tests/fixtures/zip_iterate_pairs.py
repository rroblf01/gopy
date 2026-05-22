def main() -> None:
    names: list[str] = ["alice", "bob", "carol"]
    ages: list[int] = [30, 25, 35]
    for name, age in zip(names, ages):
        print(name, age)
    pairs = list(zip(names, ages))
    for p in pairs:
        print(p[0], "-", p[1])


if __name__ == "__main__":
    main()
