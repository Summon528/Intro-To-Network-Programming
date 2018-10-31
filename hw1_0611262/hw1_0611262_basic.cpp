#include <algorithm>
#include <array>
#include <cstdio>
#include <list>
#include <memory>
#include <mutex>
#include <queue>
#include <thread>

using namespace std;

struct Player {
    int id, wake, play_round, sleep, total;
    Player(int a, int b, int c, int d, int e)
        : id{a}, wake{b}, play_round{c}, sleep{d}, total{e} {}
};

struct Machine {
    shared_ptr<Player> player;
    int play_time = 0;
};

int main(int argc, char** argv) {
    using player_ptr = shared_ptr<Player>;
    auto file = fopen(argv[1], "r");
    int G, N;
    fscanf(file, "%d%d", &G, &N);

    list<player_ptr> L;

    for (int i = 0; i < N; i++) {
        int wake, play_round, sleep, total;
        fscanf(file, "%d%d%d%d", &wake, &play_round, &sleep, &total);
        L.push_back(make_shared<Player>(i + 1, wake, play_round, sleep, total));
    }

    array<Machine, 1> machines;
    queue<player_ptr> Q;
    vector<thread> vt;
    int clk = 0, curr_G = 0;

    while (!L.empty() || any_of(machines.begin(), machines.end(),
                                [](auto& i) { return i.player != nullptr; })) {
        vt.clear();
        mutex G_lock;
        for (int i = 0; i < machines.size(); i++) {
            auto machine = machines.begin() + i;
            vt.push_back(thread([machine, G, i, clk, &L, &curr_G, &G_lock]() {
                auto& player = machine->player;
                if (player != nullptr) {
                    machine->play_time++;
                    player->total--;
                    G_lock.lock();
                    curr_G++;
                    if (machine->play_time == player->play_round ||
                        player->total == 0 || curr_G == G) {
                        printf("%d %d finish playing ", clk, player->id);
                        if (player->total && curr_G != G) {
                            G_lock.unlock();
                            puts("NO");
                            player->wake = clk + player->sleep;
                            L.push_back(player);
                        } else {
                            curr_G = 0;
                            G_lock.unlock();
                            puts("YES");
                        }
                        player = nullptr;
                        machine->play_time = 0;
                    }
                    G_lock.unlock();
                }
            }));
        }

        for_each(vt.begin(), vt.end(), [](auto& t) { t.join(); });

        vt.clear();
        mutex Q_lock;
        int idle = count_if(machines.begin(), machines.end(),
                            [](auto x) { return x.player == nullptr; });
        for (auto player = L.begin(); player != L.end(); player++) {
            vt.push_back(thread([clk, player, idle, &Q, &machines, &Q_lock]() {
                if (player->get()->wake == clk) {
                    Q_lock.lock();
                    int Qsize = Q.size();
                    Q.push(*player);
                    Q_lock.unlock();
                    if (idle <= Qsize) {
                        printf("%d %d wait in line\n", clk, player->get()->id);
                    }
                }
            }));
        }

        for_each(vt.begin(), vt.end(), [](auto& t) { t.join(); });

        auto rm_t = thread([&L, clk]() {
            L.remove_if([clk](auto& x) { return x->wake == clk; });
        });

        vt.clear();
        for (auto i = 0; i < machines.size(); i++) {
            auto machine = machines.begin() + i;
            vt.push_back(thread([clk, machine, i, &Q, &Q_lock]() {
                Q_lock.lock();
                if (!Q.empty() && machine->player == nullptr) {
                    auto front = Q.front();
                    Q.pop();
                    Q_lock.unlock();
                    printf("%d %d start playing\n", clk, front->id);
                    machine->player = front;
                }
                Q_lock.unlock();
            }));
        }
        for_each(vt.begin(), vt.end(), [](auto& t) { t.join(); });

        if (all_of(machines.begin(), machines.end(),
                   [](auto& m) { return m.player == nullptr; })) {
            curr_G = 0;
        }
        clk++;
        rm_t.join();
    }
}
