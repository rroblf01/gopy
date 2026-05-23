def main() -> None:
    items: list = [42, "hello", 3.14, True]
    for x in items:
        if isinstance(x, str):
            print("str:", x)
        elif isinstance(x, bool):
            print("bool:", x)
        elif isinstance(x, int):
            print("int:", x)
        elif isinstance(x, float):
            print("float:", x)
        else:
            print("?")


if __name__ == "__main__":
    main()
