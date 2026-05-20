def main() -> None:
    pi = 3.14159
    n = 42
    name = "ada"
    print(f"[{n:5d}]")
    print(f"[{n:05d}]")
    print(f"[{pi:.2f}]")
    print(f"[{pi:8.2f}]")
    print(f"[{name:>10}]")
    print(f"[{name:<10}]")
    print(f"[{name:^10}]")
    print(f"[{n:x}]")
    print(f"[{n:08x}]")
    print(f"[{n:b}]")


if __name__ == "__main__":
    main()
