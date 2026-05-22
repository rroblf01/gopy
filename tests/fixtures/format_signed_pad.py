def main() -> None:
    # Format float with comma separator (skip - not supported)
    # Format with sign
    print(f"{42:+d}")
    print(f"{-42:+d}")
    print(f"{3.14:+.2f}")
    # Format with thousands separator skipped
    # Hex with leading zeros
    print(f"{255:04x}")
    print(f"{15:04X}")


if __name__ == "__main__":
    main()
