#include <iostream>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/wait.h>

using namespace std;

#define BUFFER_SIZE 21 // 20 bytes para os caracteres + 1 byte para o terminador de string '\0'.

int generateRandomNumber(int old_n)
{
    int delta = (rand() % 100) + 1;
    return (old_n) + delta;
}

int stringToInt(const char *str)
{
    int result = 0;
    for (int i = 0; i < BUFFER_SIZE - 1; i++)
    {
        if (str[i] >= '0' && str[i] <= '9')
        {
            result = result * 10 + (str[i] - '0');
        }
        else
        {
            break;
        }
    }
    return result;
}

void writeInBuffer(char *buffer, int number)
{
    snprintf(buffer, BUFFER_SIZE, "%020d", number);
}

bool isPrime(int n)
{
    if (n <= 1)
        return 0;
    for (int i = 2; i <= n / 2; i++)
    {
        if (n % i == 0)
            return 0;
    }
    return 1;
}

int main(int argc, char *argv[])
{
    if (argc != 2)
    {
        cerr << "Uso correto: " << argv[0] << " <quantidade_de_numeros>" << endl;
        return 1;
    }

    int n = atoi(argv[1]);
    if (n <= 0)
    {
        cerr << "A quantidade de números deve ser maior que zero." << endl;
        return 1;
    }

    srand(time(NULL));

    // fd[0] = read end (leitura)
    // fd[1] = write end (escrita)
    int fd[2];

    if (pipe(fd) == -1)
    {
        perror("Erro ao criar o pipe");
        return 1;
    }

    pid_t pid = fork();

    if (pid < 0)
    {
        perror("Erro no fork");
        return 1;
    }

    // Produtor (fork retorna um pid > 0 para o processo pai).
    if (pid > 0)
    {
        close(fd[0]); // Aqui fecho a ponta de leitura, porque o Pai só escreve no cano, não tem interesse em ler nada.

        int current_number = 1;
        char buffer[BUFFER_SIZE];
        cout << "\n[PRODUTOR]: Iniciando o trabalho de escrita no cano...\n"
             << endl;

        while (n-- > 0)
        {
            current_number = generateRandomNumber(current_number);

            writeInBuffer(buffer, current_number);
            printf("[PRODUTOR]: Número gerado: %d, Enviado como: '%s'\n", current_number, buffer);
            write(fd[1], buffer, BUFFER_SIZE);
        }
        writeInBuffer(buffer, n);
        write(fd[1], buffer, BUFFER_SIZE); // Envio o '0' para sinalizar que não tem mais números a serem enviados.

        close(fd[1]);
        // Boa prática: O Pai espera o Filho terminar o trabalho dele antes de morrer
        wait(NULL);
        printf("[PRODUTOR]:  Meu filho terminou, encerrando também.\n");
    }
    else if (pid == 0) // Consumidor (para o novo processo, o fork() devolve exatamente 0).
    {
        close(fd[1]); // De forma analoga, o Filho fecha a ponta de escrita, porque ele só tem interesse em ler do cano, não tem interesse em escrever nada.

        cout << "\n[CONSUMIDOR]: Iniciando o trabalho de leitura do cano...\n"
             << endl;

        while (true)
        {
            char buffer[BUFFER_SIZE];
            read(fd[0], buffer, BUFFER_SIZE);

            int number = stringToInt(buffer);

            if (number != 0)
            {
                cout << "[CONSUMIDOR]: " << "O número fornecido (" << number << ") é primo? " << (isPrime(number) ? "Sim" : "Não") << endl;
            }
            else
            {
                printf("\n[CONSUMIDOR]: Trabalho concluído, encerrando.\n");
                break;
            }
        }
        close(fd[0]); // Quando o read() receber 0, ele sai do loop e fecha a própria ponta de leitura
    }

    return 0;
}