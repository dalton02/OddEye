//go:build linux

package proc

/*
#include <signal.h>
static void reset_sigusr1(void) { signal(SIGUSR1, SIG_DFL); }
*/
import "C"

// QuietWebKit devolve o SIGUSR1 ao padrão antes de o WebKitGTK subir. O runtime do Go instala handler
// em todos os sinais; o JavaScriptCore usa o SIGUSR1 no GC e, ao achá-lo ocupado, escreve no stderr
// "Overriding existing handler for signal 10". É só um aviso, mas lançadores e gerenciadores de
// arquivo mostram o stderr como "Command failed". O Go não usa o SIGUSR1, então soltá-lo é seguro.
// (A variável JSC_SIGNAL_FOR_GC não resolve: qualquer outro sinal também já tem handler do Go.)
func QuietWebKit() { C.reset_sigusr1() }
