export interface Point { x: number; y: number }
export interface Field { name: string; type?: string; primary?: boolean }
export interface Item { id: string; name: string; fields?: Field[]; methods?: string[]; shape?: string; role?: string }
export interface Relation { id: string; source: string; target: string; label?: string; kind?: string; source_cardinality?: string; target_cardinality?: string }
export type DiagramType = 'er' | 'table' | 'usecase' | 'flowchart' | 'sequence' | 'class' | 'architecture' | 'structure'
export type DiagramTheme = 'blue' | 'mono' | 'sage'
export const diagramThemes: { id: DiagramTheme; label: string; color: string }[] = [{ id: 'blue', label: '蓝图', color: '#586fe8' }, { id: 'mono', label: '论文', color: '#343b46' }, { id: 'sage', label: '青绿', color: '#6a8c7c' }]
export interface Model { entities?: Item[]; actors?: Item[]; use_cases?: Item[]; classes?: Item[]; participants?: Item[]; nodes?: Item[]; relations?: Relation[]; messages?: Relation[]; system_name?: string }
export interface Diagram { schema_version: string; id: string; type: DiagramType; title: string; description?: string; model: Model; view: { theme: string; direction: string; positions: Record<string, Point> } }
export interface Project { schema_version: string; name: string; diagrams: Diagram[]; updated_at?: string }
export interface Issue { path: string; message: string; severity: 'error' | 'warning' }
export interface SceneNode { id: string; x: number; y: number; width: number; height: number; shape: string; title: string; lines?: string[]; methods?: string[]; fill: string; stroke: string; movable: boolean }
export interface SceneEdge { id: string; source?: string; target?: string; points: Point[]; label?: string; label_position?:Point; label_angle?:number; source_label?: string; source_label_position?:Point; target_label?: string; target_label_position?:Point; dashed: boolean; arrow: string }
export interface Scene { width: number; height: number; nodes: SceneNode[]; edges: SceneEdge[]; type: string; title: string; theme: string }
export interface RenderResult { validation: { valid: boolean; issues: Issue[] }; scene: Scene; svg: string; node_svgs: Record<string, string> }
export interface Session { project: Project; path: string; recovered: boolean; recovery_error?: string;revision?:string }
export interface WorkspaceEntry {name:string;path:string;kind:'folder'|'file';diagram_type?:DiagramType;children?:WorkspaceEntry[]}
export interface WorkspaceTree {root:string;entries:WorkspaceEntry[]}
export interface WorkspaceDocument {path:string;diagram:Diagram;revision:string}
export const kinds: Record<DiagramType, { label: string }> = {
  er: { label: '概念 ER 图' },
  table: { label: '表关系图' },
  usecase: { label: '用例图' },
  flowchart: { label: '流程图' },
  sequence: { label: '时序图' },
  class: { label: '类图' },
  architecture: { label: '架构图' },
  structure: { label: '功能结构图' },
}
export function itemsOf(d: Diagram): Item[] { const m = d.model; return [...(m.entities ?? []), ...(m.actors ?? []), ...(m.use_cases ?? []), ...(m.classes ?? []), ...(m.participants ?? []), ...(m.nodes ?? [])] }
export const clone = <T>(x: T): T => JSON.parse(JSON.stringify(x))

export function newDiagram(type:DiagramType,title:string):Diagram {
 const fields:Field[]=[{name:'id',type:'bigint',primary:true},{name:'name',type:'varchar(100)'}]
 let model:Model
 if(type==='er'||type==='table')model={entities:[{id:'entity-1',name:title,fields}],relations:[]}
 else if(type==='class')model={classes:[{id:'class-1',name:title,fields,methods:[]}],relations:[]}
 else if(type==='usecase')model={system_name:title,actors:[{id:'actor-1',name:'参与者'}],use_cases:[{id:'usecase-1',name:'用例1'}],relations:[{id:'relation-1',source:'actor-1',target:'usecase-1',kind:'association'}]}
 else if(type==='sequence')model={participants:[{id:'sender',name:'发起方'},{id:'receiver',name:'接收方'}],messages:[{id:'message-1',source:'sender',target:'receiver',label:'请求',kind:'call'}]}
 else if(type==='flowchart')model={nodes:[{id:'start',name:'开始',shape:'terminal'},{id:'finish',name:'结束',shape:'terminal'}],relations:[{id:'relation-1',source:'start',target:'finish'}]}
 else model={nodes:[{id:'node-1',name:title,...(type==='architecture'?{shape:'service'}:{})}],relations:[]}
 return {schema_version:'1.0',id:type+'-new',type,title,model,view:{theme:'blue',direction:['er','table','class','architecture'].includes(type)?'LR':'TB',positions:{}}}
}
