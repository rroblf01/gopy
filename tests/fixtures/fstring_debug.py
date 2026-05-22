def main() -> None:
    name = "World"
    s = f"Hello, {name}!"
    print(s)
    s2 = f"{name=}"
    print(s2)
    n = 42
    s3 = f"{n=}"
    print(s3)


if __name__ == "__main__":
    main()
