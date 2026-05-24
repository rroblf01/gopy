def main() -> None:
    name: str = "hi"
    width: int = 8
    prec: int = 2
    x: float = 3.14159

    print(f"|{name:>{width}}|")
    print(f"|{name:<{width}}|")
    print(f"|{name:^{width}}|")
    print(f"{x:.{prec}f}")
    print(f"{x:.{prec + 1}f}")


if __name__ == "__main__":
    main()
