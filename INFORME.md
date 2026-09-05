# Informe de Trabajo Practico Nivelador

## Arquitectura
La solucion es un sistema cliente-servidor distribuido, donde cada cliente representa una agencia, identificados por su agency-id, y un solo servidor que escucha por conexiones de clientes. Los clientes envian apuestas que leen de los archivos CSV de sus respectivos `INPUT_FILE` en batches de tamaño configurable por `BATCH_SIZE` a traves de conexiones TCP. El servidor recibe estas apuestas de forma concurrente, enviando un `ACK` si recibe correctamente (o `NACK` en caso contrario) y persiste las apuestas dentro del archivo `SERVER_BETS`. Luego de esperar una minima cantidad de agencias, configurable desde `AGENCY_QUORUM_MIN`, decide los ganadores y envia los correspondientes a cada agencia. Los clientes guardan a estos ganadores en sus respectivos `OUTPUT_FILE`.


## Protocolo de comunicacion
El primer cambio en el protocolo se realizo sobre las funciones de `recv_all` y `send_all` tanto para los clientes como el servidor. Para usarlas primitivas Read/recv y Write/send y garantizar la transferencia total de todos los bytes que se quieren enviar, incluso cuando las primitivas realizan lecturas/escrituras parciales, se hicieron los siguientes cambios:

`Send`: Mantiene un contador de bytes escritos. Mientras la cantidad sea menor que el tamaño del mensaje que se quiere enviar, vuelve a intentar el envio desde el primer byte pendiente.   

`Read`: Recibe como parametro una cantidad de bytes esperados. Cada lectura se agrega al resultado y actualiza el contador hasta llegar al tamaño requerido.

### Mensaje

Para poder reconocer los limites de cada mensaje, se implemento un protocolo de mensajes con longitud de paquete de esquema mixto. Cada mensaje comienza con un header de longitud fija de 4 bytes, un payload de longitud variable, con campos internos delimitados con `,` y `\n`.

```text
┌──────────────────────────────────────────┐
│ Header = 4 Bytes                         │
│ Tamaño del payload                       │
├──────────────────────────────────────────┤
│                                          │
│ Payload: Contenido del mensaje           │
│                                          │
└──────────────────────────────────────────┘
```

El tamaño almacenado en el header corresponde a la cantidad de bytes del payload. Por lo tanto, el receptor el mensaje en dos etapas: primero los 4 bytes para el header e interpretar el tamaño, y usando ese valor lee esa cantidad de bytes para el payload.

El header se codifica usando Big Endian, enviando el byte mas significativo primero. Tanto el cliente como el servidor lo leen usando ese formato e interpretandolo como unsigned.

### Tipos de mensajes
Todos los mensajes utilizan la estructura descrita anteriormente, pero sirven diferentes propositos respecto a la comunicacion entre el emisor y el receptor.

`Batch de apuestas`: Mensaje enviado desde el cliente al servidor con una o varias apuestas. Cada apuesta divide sus campos con el delimitador `,`. Cada apuesta se divide entre si con el delimitador `\n`.

`ACK`: Mensaje que envia el servidor para confirmar que el lote fue leido y procesado correctamente.

`NACK`: Mensaje que envia el servidor para informar que el lote no fue leido o procesado correctamente

`END`: El cliente lo envia cuando termino de enviar las apuestas de su agencia. El servidor lo envia al terminar de enviar una lista de ganadores.

`Winner`: Datos que envia el servidor sobre un ganador de una agencia. Cada ganador divide sus campos con el delimitador `,`.

### Uso eficiente de memoria
Inicialmente se construia un nuevo mensaje para cada batch desde el cliente para enviar al servidor. Esto generaba un elevado uso de memoria, por lo que se opto reutilizar un `[]byte` para construir los batches y reducir la memoria usada en el sistema.

Al crear un mensaje se reservan primero los cuatro bytes correspondientes al header. Luego, se agregan las apuestas directamente al mismo slice. Cuando el batch esta completo, se escribe el tamaño del payload en los primeros cuatro bytes y se envia el mensaje. Al recibir la confirmacion del servidor, el slice se reutiliza. Para esto se utilizan las funciones `ReuseMessage` y `NewMessage` para separar la capa de negocio y la capa de comunicacion, sin exponer el `HEADER_SIZE` de forma innecesaria al archivo de client o bets.


## Concurrencia del servidor

El servidor utiliza un thread por cada conexion con un cliente. El thread principal se encarga de crear el socket, aceptar conexiones y crear un thread por cada cliente. Ademas, se guarda un listado de los threads activos y coordina su finalizacion para asegurarse el correcto cierre de FDs.

Cada thread con el cliente se encarga de recibir los batches, deserializar y almacenar las apuestas, esperar al quorum para calcular a los ganadores y enviar la respuesta. Una vez finalizado se encarga de cerrar su socket.

### Sincronizacion:

El acceso a `Lottery` esta protegido mediante un Mutex, ya que tanto el metodo `store_bets` como `load_bets` no son thread safe. Este lock protege tanto las escrituras del store y las lecturas del load.

Ademas se utiliza una condvar para la espera del quorum. Cuando una agencia envia `END`, su thread incrementa el contador de agencias finalizadas dentro de la condicion. Si se alcanzo el quorum, ejecuta `notify_all` para despertar a todos los threads que estaban esperando. Si no, queda esperando usando wait_for con la condicion de quorum alcanzado, la cual deja al thread bloqueado y cede el procesador hasta recibir la anterior notificacion (o un spurious wake up, pero esta contemplado verificando la condicion nuevamente). Los nuevos clientes que lleguen no van a bloquearse ya que la condicion se sigue cumpliendo. 

### Graceful shutdown

El cliente espera la señal de terminacion de forma concurrente a la comunicacion con el servidor. Para coordinar la goroutine con el thread principal, se utilizan thres channels:

`signalChannel`: recibe la señal de `SIGTERM` enviada por el sistema operativo.

`terminated`: informa al handler que el Run principal termino.

`handlerEnded`: permite que el thread principal espere la finalizacion del handler.

Al recibirla, registra que se encuentra terminando y cierra la conexion activa. Esto permite desbloquear cualquier operacion de lectura o escritura que estaba esperando sobre el socket y continuar con la finalizacion del programa. El estado de terminacion se comparte de manera segura mediante un booleano atomico.

El servidor utiliza un evento compartido para indicar que recibio el `SIGTERM`. El socket de escucha tiene un timeout que permite al thread principal comprobar si este evento paso aun si no llegan nuevas conexiones. Cuando empieza la terminacion, despierta a los threads que se encuentran esperando el quorum para que se desbloqueen y cierra los sockets de los clientes. Termina por esperar la finalizacion de todos los thread mediante un `join` y cierra el socket principal.