def main() -> None:
    print(f"{1234567:,}")
    print(f"{1234567:_}")
    print(f"{1234567.5:,.2f}")
    print(f"{-1234567:,}")
    print(f"{1000:,d}")
    print(f"{0.05:.1%}")
    print(f"{0.5:%}")
    print(f"{1.234:.3%}")
    print("{:,.2f}".format(98765.4321))
    print("{:_d}".format(1000000))


if __name__ == "__main__":
    main()
