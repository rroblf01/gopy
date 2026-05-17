class Animal:
    def __init__(self, name: str) -> None:
        self.name = name

    def greet(self) -> str:
        return f"hello {self.name}"


class Dog(Animal):
    def __init__(self, name: str, breed: str) -> None:
        super().__init__(name)
        self.breed = breed

    def describe(self) -> str:
        return f"{self.greet()} ({self.breed})"


def main() -> None:
    d: Dog = Dog("rex", "lab")
    print(d.greet())
    print(d.describe())
    print(d.name)
    print(d.breed)


if __name__ == "__main__":
    main()
