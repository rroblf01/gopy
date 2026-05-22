def main() -> None:
    print(f"{42:5d}")
    print(f"{42:>5d}")
    print(f"{42:<5d}|")
    print(f"{42:^7d}|")
    print(f"{42:05d}")
    print(f"{3.14:8.4f}")
    print(f"{3.14:.2f}")
    print(f"{255:x}")
    print(f"{255:X}")
    print(f"{255:#x}")
    print(f"{255:#X}")
    print(f"{255:o}")
    print(f"{42:b}")
    print(f"{42:08b}")


if __name__ == "__main__":
    main()
