package main
/**
* ---------------------IMPORTACIONES-------------------
*/
import (
    "encoding/json"
    "github.com/gorilla/mux"
    "html/template"
    "net/http"
    "github.com/shirou/gopsutil/process"
    "github.com/guillermo/go.procmeminfo"
    "github.com/shirou/gopsutil/cpu"
    "math"
    "log"
    "fmt"
    "regexp"
    "strconv"
)
var router = mux.NewRouter()
/**
* ---------------------ESTRUCTURAS-------------------
*/
type USUARIOS struct{
  Usuario string
  Contrasenia string
}

type PROCESO struct{
  PID int32
  Usuario string
  Estado string
  Memoria float32
  Nombre string
  Proceso *process.Process
}

type INFO_PROCESO struct{
  TotalProcesos int
  TotalEjecucion int
  TotalSuspendidos int
  TotalDetenidos int
  TotalZombie int
  ListaProcesos []PROCESO
}

/**
* ------------------VARIABLES GLOBALES-----------------
*/
var CantRun, CantSleep, CantStop, CantZombie int
var ListaProcesos [] PROCESO

/**
* -----------------------LOGIN-----------------------
*/

func Login(w http.ResponseWriter, r *http.Request){
  NuevoUsuario := USUARIOS{
    Usuario:"admin",
    Contrasenia:"admin",
  }
  tmpl := template.Must(template.ParseFiles("login.html"))
  tmpl.Execute(w, NuevoUsuario)
}

/**
* -----------------MENU PRINCIPAL---------------------
*/

func Home(w http.ResponseWriter, r *http.Request){
  //http.ServeFile(w, r, "index.html")
  // * * * Reiniciar variables para los procesos * * * 
  InitVarProcesos()
  // * * * Obtener los procesos accediendo al metodo que se importo * * * 
  ListaProcesosActual, err := process.Processes()
  dealwithErr(err)

  for _, ProcesoIterado := range ListaProcesosActual{
    usuario, _ := ProcesoIterado.Username()
    estado, _ := ProcesoIterado.Status()
    memoria, _ := ProcesoIterado.MemoryPercent()
    nombre, _ := ProcesoIterado.Name()

    ProcesoActual1 := PROCESO{
      PID: ProcesoIterado.Pid,
      Usuario: usuario,
      Estado: GETEstadoProceso(estado),
      Memoria: memoria,
      Nombre: nombre,
      Proceso: ProcesoIterado,
    }
    ListaProcesos = append(ListaProcesos, ProcesoActual1)
  }
  // Informacion total de los procesos
  Info := INFO_PROCESO {
		TotalProcesos : len(ListaProcesos),
		TotalEjecucion : CantRun,
		TotalSuspendidos : CantSleep,
		TotalDetenidos : CantStop,
		TotalZombie : CantZombie,
		ListaProcesos : ListaProcesos,
  }

  tmpls, errortamplate := template.ParseFiles("index.html")
  if errortamplate != nil {
    http.Error(w, errortamplate.Error(), http.StatusInternalServerError)
    return
  }

  if errortamplate := tmpls.Execute(w, Info); errortamplate != nil {
    http.Error(w, errortamplate.Error(), http.StatusInternalServerError)
  }
}

/**
*	[R=Running] [S=Sleep] [T=Stop] [I=Idle] [Z=Zombie] [W=Wait] [L=Lock]
*/
func GETEstadoProceso(charindex string)(string){
  switch charindex {
    case "R":
      CantRun+=1
      return "Running"
    case "S":
      CantSleep+=1
      return "Sleep"
    case "T":
      CantStop+=1
      return "Stop"
    case "Z":
      CantZombie+=1
      return "Zombie"
    case "I":
      return "Idle"
    case "W":
      return "Wait"
    case "L":
      return "Lock"
    default:
      return "NaN"
  }
}

/**
* KILL Procesos
* func killProceso(w http.ResponseWriter, r *http.Request, info INFO_PROCESO){
*/
var validacionPath = regexp.MustCompile("^/(kill)/([a-zA-Z0-9]+)$")
func MatarProceso(w http.ResponseWriter, r *http.Request, pid string) {
	for i := 0; i < len(ListaProcesos); i++ {
    //Entra Si encuentro el proceso que se quiere terminar
		if string(strconv.Itoa(int(ListaProcesos[i].PID))) == pid{ 
			log.Println("Te encontre proceso: ", pid)
			ListaProcesos[i].Proceso.Kill()
			break
		}	
	}
  Home(w, r)
}

func CustomizarPath(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
  
  return func(w http.ResponseWriter, r *http.Request) {
    m := validacionPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
      http.NotFound(w, r)
      return
    }
		fn(w, r, m[2])
	}
}
/**
* ------------------------CPU------------------------
*/
func indexCPU(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "cpu.html")
}

func datosCPU(w http.ResponseWriter, r *http.Request) {
	vmStat, err := cpu.Percent(0,false);
  dealwithErr(err)
  
	type CPU struct {
		Porcentaje float64
  }
  
	datos := CPU{Porcentaje : math.Floor(vmStat[0]*100)/100}

	datos_json , err := json.Marshal(datos)
	dealwithErr(err)

	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(datos_json)
}

/**
* ------------------------RAM------------------------
*/
func indexmemoriaRAM(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "ram.html")
}

func datosRAM(w http.ResponseWriter, r *http.Request) {
	var total, consumida, porcentaje_consumo, megabytes float64
  megabytes = 1024 * 1024
  
	meminfo := &procmeminfo.MemInfo{}
  meminfo.Update()
  
	total = (float64) (meminfo.Total()) / megabytes
	consumida = (float64) (meminfo.Used()) / megabytes
	porcentaje_consumo = ((consumida * 100) / total)
  

	type MEMORIA struct {
		Total float64
		Consumida float64
		Porcentaje float64
	}

	datos := MEMORIA{Total : total, Consumida : consumida, Porcentaje : porcentaje_consumo}

	datos_json , err := json.Marshal(datos)
  dealwithErr(err)
  
  w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(datos_json)
}

/**
* ------------------------MAIN------------------------
*/
func main() {
    puerto := ":8082"
    router.HandleFunc("/", Login)
    router.HandleFunc("/Home", Home)
    http.HandleFunc("/kill/", CustomizarPath(MatarProceso))

    router.HandleFunc("/RAM", indexmemoriaRAM)
    router.HandleFunc("/datosRAM", datosRAM)

    router.HandleFunc("/CPU", indexCPU)
    router.HandleFunc("/datosCPU", datosCPU)
    
    
    fmt.Println("------ Iniciando Servidor ------");
    http.Handle("/", router)
    log.Println("Servidor listo en la url = http://localhost:8082/ | Puerto", puerto)
    log.Fatal(http.ListenAndServe(puerto, nil))
}
/**
* ------------------------UTILES------------------------
*/
func InitVarProcesos(){
  CantRun = 0
  CantSleep = 0
  CantStop = 0
  CantZombie = 0
  ListaProcesos = nil
}

func dealwithErr(err error) {
	if err != nil {
		fmt.Println(err)
		//os.Exit(-1)
	}
}

func bToMb(b uint64) uint64 {
  return b / 1024 / 1024
}