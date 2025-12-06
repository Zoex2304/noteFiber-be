Before saving:
Checklist appears with the correct visuals (icons present, clear interactions)
Table appears neat with borders and column structure
Formatting such as underline, bold, and strike appear normal
After saving and re-rendering from the backend data:
Checklist changes to raw text such as [x] Start writing
Table is still there but the styling and visual structure have changed
Underline, strike, and some other formats appear inconsistent
Some checkboxes even become empty without visual icons
Meaning:
What is stored in the backend and what is rendered back to the editor is not in the same format representation.
2. Important Note: This is NOT a Backend Structure Issue
This is NOT an issue with:
Backend endpoint
Database structure
JWT, OAuth, or API authentication logic
Or CORS issues
Your backend only does:
Receive payload → save it
Send back saved data
The backend is not responsible for the visual checklist, tables, or editor formatting.
This issue is 100% in the Frontend layer + Lexical Serialization Logic.
Technically, this issue occurs because:
The Lexical state before saving is EditorState (Lexical-specific JSON)
When saving, what is likely sent to the backend is:
Converted HTML
or plain text
or Markdown
When reloading, that data:
is not converted back to the Lexical node format
and is then rendered as regular static content
As a result:
Checklist Node → changes to a string [x]
Table Node → loses its node schema
DecoratorNode (icons, underlines, UI) → is no longer rendered
3. The Most Likely Technical Root Cause
One (or a combination) of the following conditions occurs in the Frontend:
When saving, you use:
$generateHtmlFromNodes
then save the HTML to the backend
However, when reloading:
The HTML is not parsed back into Lexical EditorState
Or:
You only save editorState.toJSON()
but when loading, you do not perform editor.setEditorState() with that JSON
So what happens is:
When loading, the editor does not recognize the content as a Lexical structural Node, but only as plain text.
4. Why does it look fine before saving, but broken after saving?
Because before saving:
The data is still completely within the Lexical Runtime EditorState
All Nodes are active: TableNode, ListNode, CheckListNode, etc.
After saving:
The data exits the editor system
It enters the backend as a string or HTML
It returns to the frontend without being converted back to Lexical Nodes
So what is displayed is only:
The passive render result
Without the interactive logic of the editor
Without visual decorators (icons, active checkboxes, etc.)
5. Is this related to the backend structure logic?
Firm answer:
No.
This has absolutely nothing to do with backend structure logic.
Your backend:
Is correctly receiving data
Is correctly storing data
Is correctly sending data back
What is not correct is:
The data format contract between the Lexical Editor and the Backend
The data rehydration process to EditorState in the frontend
6. Narrative Instructions for the Agent (Ready to Send to the Team)
Please send the following narrative to the agent:
Currently, the issue with the formatting of checklists, tables, and underlines changing after the save process does not originate from the backend. The backend only stores and returns data; it does not control how the editor is rendered. The main issue lies in the frontend Lexical serialization–deserialization flow.
Before saving, the content is still in Lexical EditorState format, so all nodes such as checklists, tables, and decorators appear normal. After saving and reloading, the data retrieved from the backend is not converted back to Lexical node format, so it is only rendered as static text or HTML.
The solution is to ensure that:
When saving, what is sent to the backend is Lexical JSON EditorState, not just HTML or plain text.
When loading, the JSON data must be parsed again using editor.setEditorState() or editor.parseEditorState().
If you are still using HTML as the storage medium, then an HTML → Lexical Nodes pipeline with an official transformer must be created. Without this, checklists will remain [x] and tables will lose their editor UI structure.
This issue is unrelated to the API, JWT, CORS, or database schema.
7. Final Conclusion
This issue is purely frontend rendering & data transformation.
Your backend is structurally correct.
What needs to be fixed is:
How to save Lexical state.
How to rehydrate Lexical state after loading.
If you'd like, I can continue with:
The correct EditorState storage schema
The JSON flow: Lexical → Backend → Rehydrate to Editor
An example architecture aligned with your Atomic Design principles (Atom → Molecule → Organism → Template)
Just let me know: proceed with the technical fixes or not.