def main() -> None:
    xs = [n for n in range(30) if n % 2 == 0 if n % 3 == 0]
    print(xs)

    ys = [n for n in range(20) if n > 5 if n < 15 if n % 2 == 1]
    print(ys)


if __name__ == "__main__":
    main()
