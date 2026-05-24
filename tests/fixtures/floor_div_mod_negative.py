def main() -> None:
    print(10 % 3)
    print(-10 % 3)
    print(10 % -3)
    print(-10 % -3)
    print(10 // 3)
    print(-10 // 3)
    print(10 // -3)
    print(-10 // -3)
    q, r = divmod(-10, 3)
    print(q, r)
    q2, r2 = divmod(10, -3)
    print(q2, r2)
    q3, r3 = divmod(-10, -3)
    print(q3, r3)
    # zero divs
    print(0 % 5)
    print(0 // 5)


if __name__ == "__main__":
    main()
