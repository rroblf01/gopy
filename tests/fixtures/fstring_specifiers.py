def main() -> None:
    name = "World"
    age = 42
    # str.format
    print("Hello, {}! You are {} years old.".format(name, age))
    # format with positional indices
    print("{0} + {1} = {2}".format(1, 2, 3))
    # f-string fancy
    pi = 3.14159
    print(f"Pi rounded: {pi:.2f}")
    print(f"Pi padded: {pi:8.3f}")
    print(f"Number: {42:5d}")
    print(f"Hex: {255:x}")
    print(f"Bin: {7:b}")


if __name__ == "__main__":
    main()
