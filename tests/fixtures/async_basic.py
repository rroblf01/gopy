import asyncio


async def greet(name: str) -> str:
    return f"hello {name}"


async def main_async() -> None:
    a = await greet("alice")
    print(a)
    b = await greet("bob")
    print(b)


def main() -> None:
    asyncio.run(main_async())


if __name__ == "__main__":
    main()
