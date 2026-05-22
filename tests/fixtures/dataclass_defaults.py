from dataclasses import dataclass


@dataclass
class Book:
    title: str
    pages: int
    price: float = 9.99


def main() -> None:
    b = Book("Python", 300)
    print(b.title, b.pages, b.price)
    c = Book("Go", 250, 19.99)
    print(c.title, c.pages, c.price)
    if b.pages < c.pages:
        print("b shorter")
    else:
        print("c shorter")


if __name__ == "__main__":
    main()
