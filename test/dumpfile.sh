import argparse
import sys
import pandas as pd

def read_parquet_file(file_path: str) -> pd.DataFrame:
    """
    Reads a Parquet file and returns its contents as a pandas DataFrame.

    Args:
        file_path (str): Path to the Parquet file.

    Returns:
        pd.DataFrame: DataFrame containing the Parquet file data.
    """
    try:
        df = pd.read_parquet(file_path, engine='pyarrow')
        return df
    except FileNotFoundError:
        print(f"Error: File '{file_path}' not found.", file=sys.stderr)
        sys.exit(1)
    except ValueError as ve:
        print(f"Error reading Parquet file: {ve}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"An unexpected error occurred: {e}", file=sys.stderr)
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(
        description="Read a Snappy-compressed Parquet file and print its contents."
    )
    parser.add_argument(
        "file_path",
        type=str,
        help="Path to the Snappy-compressed Parquet file."
    )
    args = parser.parse_args()

    df = read_parquet_file(args.file_path)

    # Option 1: Adjust display settings
    pd.set_option('display.max_rows', None)
    pd.set_option('display.max_columns', None)
    pd.set_option('display.width', None)
    pd.set_option('display.max_colwidth', None)
    print(df)

    # Option 2: Use to_string()
    # print(df.to_string())

if __name__ == "__main__":
    main()
