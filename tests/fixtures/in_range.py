def main() -> None:
    print(5 in range(10))
    print(10 in range(10))
    print(-1 in range(10))
    print(3 in range(2, 8))
    print(1 in range(2, 8))
    print(4 in range(0, 10, 2))
    print(5 in range(0, 10, 2))
    print(8 in range(10, 0, -2))
    print(7 in range(10, 0, -2))
    print(5 not in range(10))
    print(15 not in range(10))


if __name__ == "__main__":
    main()
