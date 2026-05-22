def main() -> None:
    for i in range(10):
        if i % 3 == 0:
            continue
        if i > 7:
            break
        print(i)
    # nested
    for i in range(3):
        for j in range(3):
            if i == j:
                continue
            print(i, j)


if __name__ == "__main__":
    main()
