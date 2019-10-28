package org.aion.avm.core.miscvisitors;

import org.aion.avm.core.ClassToolchain;
import org.objectweb.asm.MethodVisitor;
import org.objectweb.asm.Opcodes;

public class MapAPIClassVisitor extends ClassToolchain.ToolChainClassVisitor {

    public MapAPIClassVisitor() {
        super(Opcodes.ASM6);
    }

    @Override
    public MethodVisitor visitMethod(int access, String name, String desc, String signature, String[] exceptions) {
        MethodVisitor mv = super.visitMethod(access, name, desc, signature, exceptions);
        return new MethodVisitor(Opcodes.ASM6, mv) {
            @Override
            public void visitMethodInsn(
                    int opcode,
                    String owner,
                    String name,
                    String descriptor,
                    boolean isInterface) {
                if (opcode==Opcodes.INVOKESTATIC &&
                        owner.equals("p/avm/Blockchain") &&
                        name.equals("avm_log") &&
                        descriptor.equals("(Lw/_p/avm/Value;Lw/_p/avm/Value;)V") &&
                        !isInterface) {
                    descriptor = "([Li/ObjectArray;Li/ObjectArray;)V";
                }
                super.visitMethodInsn(opcode, owner, name, descriptor, isInterface);
            }
        };
    }
}
