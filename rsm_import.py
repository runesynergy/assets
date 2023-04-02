bl_info = {
    "name": "Rune Synergy",
    "author": "Dane",
    "version": (0, 1),
    "blender": (3, 4, 1),
    "description": "",
    "warning": "",
    "doc_url": "",
    "category": "Import-Export",
}

import bpy
import bmesh
import math

def read_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        data = f.read()
    return data.splitlines()

def parse_flat(tokens):
    return int(tokens[1]), []

def parse_line(line, destination):
    destination.append(tuple(map(int, line.split())))
    return

def process_lines(lines):
    sections = {}
    destination, elements = None, None

    for line in lines:
        tokens = line.split()

        if destination is None:
            elements, destination = parse_flat(tokens)
            sections[tokens[0]] = destination
        else:
            parse_line(line, destination)
            elements = elements - 1
            if elements == 0:
                destination, elements = None, None

    return sections

def read_some_data(context, filepath):
    lines = read_file(filepath)
    sections = process_lines(lines)
    
    mesh = bpy.data.meshes.new("Mesh")
    obj = bpy.data.objects.new("Object", mesh)
    
    bpy.context.collection.objects.link(obj)
    bpy.context.view_layer.objects.active = obj
    
    vertices = []
    faces = []
    
    for v in sections["vertices"]:
        vertices.append(v[1:4])
        
    for f in sections["faces"]:
        faces.append(f[2:5])
    
    bm = bmesh.new()
    
    for vertex in vertices:
        bm.verts.new(vertex)
    bm.verts.ensure_lookup_table()
    
    for face in faces:
        print(face)
        bm.faces.new([bm.verts[i] for i in face])
    bm.faces.ensure_lookup_table()
    
    bm.to_mesh(mesh)
    bm.free()
    
    mesh.update()
    
    obj.rotation_euler = [math.radians(-90), 0, math.radians(90)]
    obj.select_set(True)
    bpy.ops.object.transform_apply()  
    
    print(sections)
    return {'FINISHED'}

# ImportHelper is a helper class, defines filename and
# invoke() function which calls the file selector.
from bpy_extras.io_utils import ImportHelper
from bpy.props import StringProperty, BoolProperty, EnumProperty
from bpy.types import Operator


class ImportRSM(Operator, ImportHelper):
    """The official Rune Synergy addon."""
    bl_idname = "runesynergy.import_rsm"  # important since its how bpy.ops.import_test.some_data is constructed
    bl_label = "Import RSM"

    # ImportHelper mixin class uses this
    filename_ext = ".rsm"

    filter_glob: StringProperty(
        default="*.rsm",
        options={'HIDDEN'},
        maxlen=255,  # Max internal buffer length, longer would be clamped.
    )

    def execute(self, context):
        return read_some_data(context, self.filepath)


# Only needed if you want to add into a dynamic menu.
def menu_func_import(self, context):
    self.layout.operator(ImportRSM.bl_idname, text="Rune Synergy (.rsm)")


# Register and add to the "file selector" menu (required to use F3 search "Text Import Operator" for quick access).
def register():
    bpy.utils.register_class(ImportRSM)
    bpy.types.TOPBAR_MT_file_import.append(menu_func_import)


def unregister():
    bpy.utils.unregister_class(ImportRSM)
    bpy.types.TOPBAR_MT_file_import.remove(menu_func_import)


if __name__ == "__main__":
    register()
