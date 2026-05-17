def greet(name: str, age: int) -> str:
    return f"hello {name}, age {age}"


def main() -> None:
    print(greet("ada", 36))
    x: int = 7
    y: int = 5
    print(f"{x} + {y} = {x + y}")


if __name__ == "__main__":
    main()
