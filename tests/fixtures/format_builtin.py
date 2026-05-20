def main() -> None:
    print(format(3.14159, ".2f"))
    print(format(42, "5d"))
    print(format(42, "05d"))
    print(format("hi", ">10"))
    print(format(255, "x"))
    print(format(255, "08x"))
    print(format(42))
    print(format("hi"))


if __name__ == "__main__":
    main()
