def main() -> None:
    total: int = 0
    for i in range(1, 1000001):
        total += i * i - i
    print(total)


if __name__ == "__main__":
    main()
