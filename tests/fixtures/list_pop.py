def main() -> None:
    xs: list[int] = [10, 20, 30, 40, 50]
    print(xs.pop())
    print(xs.pop(0))
    print(xs.pop(-1))
    for v in xs:
        print(v)


if __name__ == "__main__":
    main()
