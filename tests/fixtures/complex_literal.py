def main() -> None:
    c = 1 + 2j
    print(c.real)
    print(c.imag)
    z = 3j
    print(z.imag)
    w = c + z
    print(w.real)
    print(w.imag)


if __name__ == "__main__":
    main()
