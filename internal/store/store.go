package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Zone struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Region string `json:"region"`
	Type string `json:"type"`
	Capacity int `json:"capacity"`
	Used int `json:"used_count"`
	Status string `json:"status"`
	Notes string `json:"notes"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"rangeland.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS zones(id TEXT PRIMARY KEY,name TEXT NOT NULL,region TEXT DEFAULT '',type TEXT DEFAULT 'production',capacity INTEGER DEFAULT 0,used_count INTEGER DEFAULT 0,status TEXT DEFAULT 'active',notes TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Zone)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO zones(id,name,region,type,capacity,used_count,status,notes,created_at)VALUES(?,?,?,?,?,?,?,?,?)`,e.ID,e.Name,e.Region,e.Type,e.Capacity,e.Used,e.Status,e.Notes,e.CreatedAt);return err}
func(d *DB)Get(id string)*Zone{var e Zone;if d.db.QueryRow(`SELECT id,name,region,type,capacity,used_count,status,notes,created_at FROM zones WHERE id=?`,id).Scan(&e.ID,&e.Name,&e.Region,&e.Type,&e.Capacity,&e.Used,&e.Status,&e.Notes,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Zone{rows,_:=d.db.Query(`SELECT id,name,region,type,capacity,used_count,status,notes,created_at FROM zones ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Zone;for rows.Next(){var e Zone;rows.Scan(&e.ID,&e.Name,&e.Region,&e.Type,&e.Capacity,&e.Used,&e.Status,&e.Notes,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Update(e *Zone)error{_,err:=d.db.Exec(`UPDATE zones SET name=?,region=?,type=?,capacity=?,used_count=?,status=?,notes=? WHERE id=?`,e.Name,e.Region,e.Type,e.Capacity,e.Used,e.Status,e.Notes,e.ID);return err}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM zones WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM zones`).Scan(&n);return n}

func(d *DB)Search(q string, filters map[string]string)[]Zone{
    where:="1=1"
    args:=[]any{}
    if q!=""{
        where+=" AND (name LIKE ?)"
        args=append(args,"%"+q+"%");
    }
    if v,ok:=filters["type"];ok&&v!=""{where+=" AND type=?";args=append(args,v)}
    if v,ok:=filters["status"];ok&&v!=""{where+=" AND status=?";args=append(args,v)}
    rows,_:=d.db.Query(`SELECT id,name,region,type,capacity,used_count,status,notes,created_at FROM zones WHERE `+where+` ORDER BY created_at DESC`,args...)
    if rows==nil{return nil};defer rows.Close()
    var o []Zone;for rows.Next(){var e Zone;rows.Scan(&e.ID,&e.Name,&e.Region,&e.Type,&e.Capacity,&e.Used,&e.Status,&e.Notes,&e.CreatedAt);o=append(o,e)};return o
}

func(d *DB)Stats()map[string]any{
    m:=map[string]any{"total":d.Count()}
    rows,_:=d.db.Query(`SELECT status,COUNT(*) FROM zones GROUP BY status`)
    if rows!=nil{defer rows.Close();by:=map[string]int{};for rows.Next(){var s string;var c int;rows.Scan(&s,&c);by[s]=c};m["by_status"]=by}
    return m
}
