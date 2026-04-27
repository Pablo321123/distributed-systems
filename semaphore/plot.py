import pandas as pd
import matplotlib.pyplot as plt
import glob
import os


# 1. Plot Execution Times
def plot_execution_times():
    if not os.path.exists("execution_times.csv"):
        print("execution_times.csv not found")
        return

    df = pd.read_csv("execution_times.csv")

    df["Threads (Np, Nc)"] = (
        "(" + df["Np"].astype(str) + "," + df["Nc"].astype(str) + ")"
    )

    plt.figure(figsize=(10, 6))

    for n_val in [1, 10, 100, 1000]:
        subset = df[df["N"] == n_val]
        plt.plot(
            subset["Threads (Np, Nc)"],
            subset["AvgTimeMs"],
            marker="o",
            label=f"N = {n_val}",
        )

    plt.title("Average Execution Time vs Thread Combinations")
    plt.xlabel("Thread Configuration (Producers, Consumers)")
    plt.ylabel("Average Execution Time (ms)")
    plt.grid(True, linestyle="--", alpha=0.7)
    plt.legend()
    plt.tight_layout()

    plt.savefig("execution_time_graph.png", dpi=300)
    print("Saved execution_time_graph.png")
    plt.close()


# 2. Plot Occupation History
def plot_occupations():
    csv_files = glob.glob("occupation_files/occupation_N*_Np*_Nc*.csv")
    if not csv_files:
        print("No occupation CSV files found.")
        return

    os.makedirs("occupation_plots", exist_ok=True)

    for file in csv_files:
        df = pd.read_csv(file)

        plt.figure(figsize=(12, 4))
        plt.plot(df["Operation"], df["Occupation"], color="purple", linewidth=0.5)

        plt.title(f"Buffer Occupation Over Time: {os.path.basename(file)}")
        plt.xlabel("Operation Number")
        plt.ylabel("Occupation (Items in Buffer)")
        plt.grid(True, linestyle="--", alpha=0.4)
        plt.tight_layout()

        plot_filename = os.path.join(
            "occupation_plots", os.path.basename(file).replace(".csv", ".png")
        )
        plt.savefig(plot_filename, dpi=150)
        plt.close()

    print(
        f"Saved {len(csv_files)} occupation plots to the 'occupation_plots' directory."
    )


if __name__ == "__main__":
    plot_execution_times()
    plot_occupations()
