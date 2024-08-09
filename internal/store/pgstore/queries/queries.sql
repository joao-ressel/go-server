-- name: GetRoom :one
SELECT
    "id", "theme"
FROM rooms
WHERE id = $1;

-- Explicação:
-- Esta consulta busca uma sala (room) específica na tabela 'rooms', com base em um 'id' fornecido como parâmetro ($1).
-- Retorna as colunas 'id' e 'theme' da sala correspondente.

-- name: GetRooms :many
SELECT
    "id", "theme"
FROM rooms;

-- Explicação:
-- Esta consulta retorna todas as salas (rooms) da tabela 'rooms'.
-- Retorna as colunas 'id' e 'theme' de todas as salas.

-- name: InsertRoom :one
INSERT INTO rooms
    ( "theme" ) VALUES
    ( $1 )
RETURNING "id";

-- Explicação:
-- Esta instrução insere uma nova sala (room) na tabela 'rooms'.
-- O tema da sala é fornecido como parâmetro ($1).
-- Após a inserção, o comando retorna o 'id' da nova sala criada.

-- name: GetMessage :one
SELECT
    "id", "room_id", "message", "reaction_count", "answered"
FROM messages
WHERE
    id = $1;

-- Explicação:
-- Esta consulta busca uma mensagem específica na tabela 'messages', com base em um 'id' fornecido como parâmetro ($1).
-- Retorna as colunas 'id', 'room_id', 'message', 'reaction_count' (contagem de reações) e 'answered' (se a mensagem foi marcada como respondida).

-- name: GetRoomMessages :many
SELECT
    "id", "room_id", "message", "reaction_count", "answered"
FROM messages
WHERE
    room_id = $1;

-- Explicação:
-- Esta consulta retorna todas as mensagens de uma sala específica, com base no 'room_id' fornecido como parâmetro ($1).
-- Retorna as colunas 'id', 'room_id', 'message', 'reaction_count' e 'answered' de todas as mensagens pertencentes à sala.

-- name: InsertMessage :one
INSERT INTO messages
    ( "room_id", "message" ) VALUES
    ( $1, $2 )
RETURNING "id";

-- Explicação:
-- Esta instrução insere uma nova mensagem na tabela 'messages'.
-- O 'room_id' e o conteúdo da mensagem são fornecidos como parâmetros ($1 e $2, respectivamente).
-- Após a inserção, o comando retorna o 'id' da nova mensagem criada.

-- name: ReactToMessage :one
UPDATE messages
SET
    reaction_count = reaction_count + 1
WHERE
    id = $1
RETURNING reaction_count;

-- Explicação:
-- Esta instrução incrementa a contagem de reações (reaction_count) de uma mensagem específica, com base no 'id' fornecido como parâmetro ($1).
-- Após a atualização, retorna o novo valor da contagem de reações.

-- name: RemoveReactionFromMessage :one
UPDATE messages
SET
    reaction_count = reaction_count - 1
WHERE
    id = $1
RETURNING reaction_count;

-- Explicação:
-- Esta instrução decrementa a contagem de reações (reaction_count) de uma mensagem específica, com base no 'id' fornecido como parâmetro ($1).
-- Após a atualização, retorna o novo valor da contagem de reações.

-- name: MarkMessageAsAnswered :exec
UPDATE messages
SET
    answered = true
WHERE
    id = $1;

-- Explicação:
-- Esta instrução marca uma mensagem como respondida, alterando o valor de 'answered' para true.
-- A mensagem é identificada pelo 'id' fornecido como parâmetro ($1).
-- Diferente das outras instruções, esta não retorna nenhum valor (uso do sufixo ':exec').
