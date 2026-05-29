import { Entity, PrimaryGeneratedColumn, Column, ManyToOne, JoinColumn, Unique, type Relation } from 'typeorm';
import { User } from './userModel.js';

@Entity('accounts')
@Unique(['provider', 'providerAccountId'])
export class Account {
  @PrimaryGeneratedColumn('uuid')
  id!: string;

  @Column()
  userId!: string;

  @Column()
  type!: string;

  @Column()
  provider!: string;

  @Column()
  providerAccountId!: string;

  @Column({ type: 'text', nullable: true })
  refresh_token?: string;

  @Column({ type: 'text', nullable: true })
  access_token?: string;

  @Column({ type: 'int', nullable: true })
  expires_at?: number;

  @Column({ nullable: true })
  token_type?: string;

  @Column({ nullable: true })
  scope?: string;

  @Column({ type: 'text', nullable: true })
  id_token?: string;

  @Column({ nullable: true })
  session_state?: string;

  // TypeORM Relation (equivalent to Prisma's onDelete: Cascade)
  @ManyToOne(() => User, (user) => user.accounts, { onDelete: 'CASCADE' })
  @JoinColumn({ name: 'userId' })
  user!: Relation<User>;
}