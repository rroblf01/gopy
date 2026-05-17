def greet(name: str, greeting: str = "hello", punct: str = "!") -> str:
    return greeting + ", " + name + punct


def main() -> None:
    print(greet("ada"))
    print(greet("ada", "hi"))
    print(greet("ada", "hi", "."))
    # Keyword arguments at the call site, in any order:
    print(greet("grace", punct="?"))
    print(greet("alan", punct="...", greeting="yo"))


if __name__ == "__main__":
    main()
